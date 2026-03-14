<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:math="http://www.w3.org/2005/xpath-functions/math"
    xmlns:f="urn:functions"
    exclude-result-prefixes="xs math"
    version="3.0">
    
    <!-- 
        show difference in principal result tree creation between creating a result tree with only text nodes or an element at the root
        variant: checking what happens when xsl:text gets a sequence of text nodes 
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
    
    <xsl:template name="xsl:initial-template">
        <xsl:text expand-text="yes">{(1 to 10)!$pfa(.)!f:create(.)}</xsl:text> 
    </xsl:template>
    
    <xsl:function name="f:create" as="text()">
        <xsl:param name="n" />
        <xsl:value-of select="$n"/>
    </xsl:function>
    
</xsl:stylesheet>