<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes" />
 <xsl:template match="doc">
  <x>
  <!-- Testing Round, xs:Decimal-->
   <?spec fo#func-round?>
    <out><xsl:value-of select="round(2.5)"/></out>
    <out><xsl:value-of select="round(-2.5)"/></out>
    <out><xsl:value-of select="round(1.25, 1)"/></out>
    <out><xsl:value-of select="round(1.27, 1)"/></out>
    <out><xsl:value-of select="round(1.224, 2)"/></out>
    <out><xsl:value-of select="round(-13.512435, 5)"/></out>
    <out><xsl:value-of select="round(199.5,-2)"/></out>
    <out><xsl:value-of select="round(199.99999999999999,4)"/></out>
    <out><xsl:value-of select="round(-13.5,0)"></xsl:value-of></out>
   <out>
    <xsl:value-of select="round(xs:decimal(sum(for $e in a return $e)), -2)"/>   
   </out>
  </x>
 </xsl:template>
</xsl:stylesheet>

