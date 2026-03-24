<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <!-- show difference in principal result tree creation between creating a result tree with only text nodes or an element at the root -->
    
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
    
    <xsl:template name="xsl:initial-template">
        <result><xsl:sequence select="(1 to 10)!$pfa(.)" /></result>
    </xsl:template>
    
</xsl:stylesheet>