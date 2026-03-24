<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:fn="http://www.w3.org/2005/xpath-functions"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:array="http://www.w3.org/2005/xpath-functions/array"
  xmlns:io="http://expath.org/ns/io"
  xmlns:task="http://expath.org/ns/task"
  exclude-result-prefixes="#all"
  version="3.0">
  
  <!-- Saxon bug #2321: Function inlining (performance check) -->
  
  <xsl:param name="realworldEl" as="element()">
    <io:realworld/>
  </xsl:param>
  
  <xsl:function name="task:create-monadic-task" as="map(*)">
    <xsl:param name="apply-fn" as="function(*)"/>
    <xsl:sequence select="map {
      'apply': $apply-fn,     
      'bind' : function($binder as function(item()*) as map(*)) as map(*) {
          let $bound-apply-fn := function($realworld) {
              let $io-res := $apply-fn($realworld)
              return $binder(fn:tail($io-res))('apply')(fn:head($io-res))
          }
          return task:create-monadic-task($bound-apply-fn)
      },
      
      'then': function($next as map(*)) as map(*) {
          let $then-apply-fn := function($realworld) {
              let $io-res := $apply-fn($realworld)
              (: NOTE: the result given by fn:head($io-res)
                       is not needed by `then`, so we can ignore it :)
              return $next('apply')(fn:head($io-res))
          }
          return task:create-monadic-task($then-apply-fn)
      },
      
      
      'fmap': function($mapper as function(item()*) as item()*) as map(*) {
          let $fmap-apply-fn := function($realworld) {
              let $io-res := $apply-fn($realworld)
              return (fn:head($io-res), $mapper(fn:tail($io-res)))
          }
          return task:create-monadic-task($fmap-apply-fn)
      },
      
      'async': function() as map(*) {
          let $async-apply-fn := function($realworld as element(io:realworld)) as item() + {
              let $exec-NO-async := $apply-fn($realworld), 
                  $async-a := function($scheduler as element(io:scheduler)) as item()+ {
                      ($scheduler, fn:tail($exec-NO-async))
                  }
              return (fn:head($exec-NO-async), $async-a)
          }
          return task:create-monadic-task($async-apply-fn)
      }
            
   }"/>
  </xsl:function>
  
  <xsl:function name="task:RUN-UNSAFE">
    <xsl:param name="task" as="map(*)"/>
    <xsl:sequence select="subsequence(
      $task('apply')($realworldEl),
      2
      )"/>
  </xsl:function>
  
  
  <!-- Example 1:  Create a task from a pure value and run it -->
  <xsl:template name="ex1">
    <xsl:sequence select="task:RUN-UNSAFE(
      task:create-monadic-task(function($realworld) {
      ($realworld, 1234)
      })
      )"/>
  </xsl:template>
  
  
</xsl:stylesheet>
