<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <!-- show difference in principal result tree creation between creating a result tree with only text nodes or an element at the root
         variant: nesting value-of should have no influence on the expand-text and the redundant spaces it returns
    -->
    
    <xsl:variable name="pfa" select="
                42!
               (let $p :=
                let $p := .
                return
                   let $p := $p
                   return 
                      let $f := function($p, $q) { $p * $q }
                      return $f(?, $p)
                return $p)" />
    
    <xsl:template name="xsl:initial-template" expand-text="yes">
       <xsl:value-of expand-text="yes">
          <xsl:value-of expand-text="no">
             <xsl:value-of expand-text="yes">
                <xsl:value-of expand-text="no">
                   <xsl:value-of expand-text="yes">
                      <xsl:value-of expand-text="yes"> <!--2x rogue space --> {(1 to 10)!$pfa(.)} <!-- 2x rogue space --> </xsl:value-of>
                   </xsl:value-of>
                </xsl:value-of>
             </xsl:value-of>
          </xsl:value-of>          
       </xsl:value-of>
    </xsl:template>
    
</xsl:stylesheet>