<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" extension-element-prefixes="xs">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes" />
 <xsl:template match="doc">
<x>
  <!-- Testing Round xs:Double -->
   <?spec fo#func-round?>
    <out><xsl:value-of select="round(12.0e0, 1)"/></out>
    <out><xsl:value-of select="round(1.154E1, 1)"/></out>
    <out><xsl:value-of select="round(1.365e1, 1)"/></out>
    <out><xsl:value-of select="round(-1.365e1, 1)"/></out>
    <out><xsl:value-of select="round(-2.5e0,0)"/></out>
    <out><xsl:value-of select="round(2.5e0,0)"/></out>
    <out><xsl:value-of select="round(2.4999e2,0)"/></out>
    <out><xsl:value-of select="round(8492.15e1, -1)"/></out>
    <out><xsl:value-of select="round(8495.551e2,-1)"/></out>
    <out><xsl:value-of select="round(8496.787e2,-2)"/></out>
    <out><xsl:value-of select="round(xs:double(45e100),-101)" /></out>
    <out><xsl:value-of select="round(xs:double(9.9e-99),99)" /></out>
    <out><xsl:value-of select="round(xs:double('0'),0)"/></out>    
    <out><xsl:value-of select="round(xs:double('NaN'),0)"/></out>
    <out><xsl:value-of select="round(xs:double(1e0 div 0e0), 0)"/></out>
    <out>
    <xsl:value-of select="round(xs:double(sum(for $e in a return xs:double($e))), 2)"/>   
    </out>
</x>
 </xsl:template>
</xsl:stylesheet>
