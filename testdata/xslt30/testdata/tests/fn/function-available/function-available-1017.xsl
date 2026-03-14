<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="3.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 xmlns:map="http://www.w3.org/2005/xpath-functions/map"
 exclude-result-prefixes="xs map"
>

<!-- Test function-available on map functions -->

  <xsl:param name="prefix" select="'map'"/>
  <xsl:param name="ns" select="'http://www.w3.org/2005/xpath-functions/map'"/>
  
  <xsl:template match="/">
    <out>
      <ok-lex>
        <a><xsl:value-of select="function-available('map:merge')"/></a> 
        <b><xsl:value-of select="function-available('map:size')"/></b> 
        <c><xsl:value-of select="function-available('map:keys')"/></c> 
        <d><xsl:value-of select="function-available('map:contains')"/></d> 
        <e><xsl:value-of select="function-available('map:get')"/></e> 
        <f><xsl:value-of select="function-available('map:put')"/></f> 
        <g><xsl:value-of select="function-available('map:entry')"/></g>
        <h><xsl:value-of select="function-available('map:remove')"/></h>
        <i><xsl:value-of select="function-available('map:for-each')"/></i>
        <z><xsl:value-of select="function-available('map:for-each-entry')"/></z>
      </ok-lex>
      <ok-lex-arity>
        <a><xsl:value-of select="function-available('map:merge', 1)"/></a> 
        <b><xsl:value-of select="function-available('map:size', 1)"/></b> 
        <c><xsl:value-of select="function-available('map:keys', 1)"/></c> 
        <d><xsl:value-of select="function-available('map:contains', 2)"/></d> 
        <e><xsl:value-of select="function-available('map:get', 2)"/></e> 
        <f><xsl:value-of select="function-available('map:put', 3)"/></f> 
        <g><xsl:value-of select="function-available('map:entry', 2)"/></g>
        <h><xsl:value-of select="function-available('map:remove', 2)"/></h>
        <i><xsl:value-of select="function-available('map:for-each', 2)"/></i>
        <z><xsl:value-of select="function-available('map:contains', 3)"/></z>
      </ok-lex-arity>
      <ok-lex-dynamic>
        <a><xsl:value-of select="function-available($prefix || ':merge')"/></a> 
        <b><xsl:value-of select="function-available($prefix || ':size')"/></b> 
        <c><xsl:value-of select="function-available($prefix || ':keys')"/></c> 
        <d><xsl:value-of select="function-available($prefix || ':contains')"/></d> 
        <e><xsl:value-of select="function-available($prefix || ':get')"/></e> 
        <f><xsl:value-of select="function-available($prefix || ':put')"/></f> 
        <g><xsl:value-of select="function-available($prefix || ':entry')"/></g>
        <h><xsl:value-of select="function-available($prefix || ':remove')"/></h>
        <i><xsl:value-of select="function-available($prefix || ':for-each')"/></i>
        <z><xsl:value-of select="function-available($prefix || ':new')"/></z>
      </ok-lex-dynamic>
      <ok-lex-dynamic-arity>
        <a><xsl:value-of select="function-available($prefix || ':merge', 1)"/></a> 
        <b><xsl:value-of select="function-available($prefix || ':size', 1)"/></b> 
        <c><xsl:value-of select="function-available($prefix || ':keys', 1)"/></c> 
        <d><xsl:value-of select="function-available($prefix || ':contains', 2)"/></d> 
        <e><xsl:value-of select="function-available($prefix || ':get', 2)"/></e> 
        <f><xsl:value-of select="function-available($prefix || ':put', 3)"/></f> 
        <g><xsl:value-of select="function-available($prefix || ':entry', 2)"/></g>
        <h><xsl:value-of select="function-available($prefix || ':remove', 2)"/></h>
        <i><xsl:value-of select="function-available($prefix || ':for-each', 2)"/></i>
        <z><xsl:value-of select="function-available($prefix || ':sandwich', 2)"/></z>
      </ok-lex-dynamic-arity>
      <ok-eqn>
        <a><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}merge')"/></a> 
        <b><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}size')"/></b> 
        <c><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}keys')"/></c> 
        <d><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}contains')"/></d> 
        <e><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}get')"/></e> 
        <f><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}put')"/></f> 
        <g><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}entry')"/></g>
        <h><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}remove')"/></h>
        <i><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}for-each')"/></i>
        <z><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}key-value-pairs')"/></z>
      </ok-eqn>
      <ok-eqn-arity>
        <a><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}merge',1)"/></a> 
        <b><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}size',1)"/></b> 
        <c><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}keys',1)"/></c> 
        <d><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}contains',2)"/></d> 
        <e><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}get',2)"/></e> 
        <f><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}put',3)"/></f> 
        <g><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}entry',2)"/></g>
        <h><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}remove',2)"/></h>
        <i><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}for-each',2)"/></i>
        <z><xsl:value-of select="function-available('Q{http://www.w3.org/2005/xpath-functions/map}for-each', 5)"/></z>
      </ok-eqn-arity>
      <ok-dynamic-eqn>
        <a><xsl:value-of select="function-available('Q{' || $ns || '}merge')"/></a> 
        <b><xsl:value-of select="function-available('Q{' || $ns || '}size')"/></b> 
        <c><xsl:value-of select="function-available('Q{' || $ns || '}keys')"/></c> 
        <d><xsl:value-of select="function-available('Q{' || $ns || '}contains')"/></d> 
        <e><xsl:value-of select="function-available('Q{' || $ns || '}get')"/></e> 
        <f><xsl:value-of select="function-available('Q{' || $ns || '}put')"/></f> 
        <g><xsl:value-of select="function-available('Q{' || $ns || '}entry')"/></g>
        <h><xsl:value-of select="function-available('Q{' || $ns || '}remove')"/></h>
        <i><xsl:value-of select="function-available('Q{' || $ns || '}for-each')"/></i>
        <z><xsl:value-of select="function-available('Q{' || $ns || '}create')"/></z>
      </ok-dynamic-eqn>
      <ok-dynamic-eqn-arity>
        <a><xsl:value-of select="function-available('Q{' || $ns || '}merge', 1)"/></a> 
        <b><xsl:value-of select="function-available('Q{' || $ns || '}size', 1)"/></b> 
        <c><xsl:value-of select="function-available('Q{' || $ns || '}keys', 1)"/></c> 
        <d><xsl:value-of select="function-available('Q{' || $ns || '}contains', 2)"/></d> 
        <e><xsl:value-of select="function-available('Q{' || $ns || '}get', 2)"/></e> 
        <f><xsl:value-of select="function-available('Q{' || $ns || '}put', 3)"/></f> 
        <g><xsl:value-of select="function-available('Q{' || $ns || '}entry', 2)"/></g>
        <h><xsl:value-of select="function-available('Q{' || $ns || '}remove', 2)"/></h>
        <i><xsl:value-of select="function-available('Q{' || $ns || '}for-each', 2)"/></i>
        <z><xsl:value-of select="function-available('Q{' || $ns || '}keys', 2)"/></z>
      </ok-dynamic-eqn-arity>
    </out>
  </xsl:template>
  

    
</xsl:stylesheet>