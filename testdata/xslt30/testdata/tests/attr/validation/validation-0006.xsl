<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:f="http://schvalid90005.com/"
exclude-result-prefixes="xs f">

<?spec xslt#validation-xsl-type?>

<!-- A contrived example to force Saxon down a particular code path, where
     validation takes place on a parentless attribute -->

<xsl:param name="p" select="''"/>
  
  
  <xsl:param name="aname" select="'a'"/>
  <xsl:param name="color" select="()"/>
  <xsl:param name="texture" select="()"/>
  
  
  
  <xsl:template match="/" name="main">
    <xsl:variable name="a" as="attribute(*)?" select="f:makeAtt($aname, $color)"/>
    
    <out>
      <xsl:if test="exists($a)">
        <xsl:value-of select="'[', string($a), ']'"/>
      </xsl:if>
    </out>
    
  </xsl:template>
  
  <xsl:function name="f:makeAtt" as="attribute(*)?">
    <xsl:param name="aname"/>
    <xsl:param name="avalue"/>
    <!-- make the function appear recursive to prevent inlining -->
    <xsl:if test="string-length($aname) > 10000">
      <xsl:sequence select="f:makeAtt(substring($aname, 10), $avalue)"/>
    </xsl:if>
    <xsl:attribute name="{$aname}" select="$color" type="xs:integer"/>
  </xsl:function>
  


</xsl:stylesheet>
