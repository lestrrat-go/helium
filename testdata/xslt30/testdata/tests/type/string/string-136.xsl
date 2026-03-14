<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0" expand-text="yes">
 
 <xsl:param name="in" select="codepoints-to-string((32, 09, 65, 13, 10, 65, 160, 32, 09))"/>

 <xsl:template name="xsl:initial-template">
  <out>   
   <xsl:variable name="replace" select="codepoints-to-string((32, 32, 65, 32, 32, 65, 160, 32, 32))"/>
   <xsl:variable name="collapse" select="codepoints-to-string((65, 32, 65, 160))"/>
   <string>{codepoint-equal(xs:string($in), $in)}</string>
   <normalizedString>{codepoint-equal(xs:normalizedString($in), $replace)}</normalizedString>
   <token>{codepoint-equal(xs:token($in), $collapse)}</token>
  </out>
 </xsl:template>


</xsl:stylesheet>