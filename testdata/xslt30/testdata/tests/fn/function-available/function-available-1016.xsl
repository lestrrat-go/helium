<?spec xslt#stylesheet-functions?>
<xsl:stylesheet version="2.0"
 xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema"
 exclude-result-prefixes="xs"
>

<!-- Test function-available on constructor functions for abstract types -->

  <xsl:param name="prefix" select="'xs'"/>
  
  <xsl:template match="/">
    <out>
      <a><xsl:value-of select="function-available('xs:NOTATION', 1)"/></a> 
      <b><xsl:value-of select="function-available('xs:anySimpleType', 1)"/></b>   
      <c><xsl:value-of select="function-available('xs:anyAtomicType', 1)"/></c> 
      <d><xsl:value-of select="function-available(concat($prefix,':NOTATION'))"/></d> 
      <e><xsl:value-of select="function-available(concat($prefix,':anySimpleType'))"/></e>   
      <f><xsl:value-of select="function-available(concat($prefix,':anyAtomicType'))"/></f> 
    </out>
  </xsl:template>
  

    
</xsl:stylesheet>