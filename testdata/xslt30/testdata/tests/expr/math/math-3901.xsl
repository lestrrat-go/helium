<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">
 <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes" />
 <xsl:template match="doc">
  <x>
  <!-- Testing fn:Round on float values -->
   <?spec fo#func-round?>
    <out><xsl:value-of select="round(xs:float(0), 0)"/></out>
    <out><xsl:value-of select="round(xs:float(12.0e0), 1)"/></out>
    <out><xsl:value-of select="round(xs:float(1.154E1), 1)"/></out>
    <out><xsl:value-of select="round(xs:float(1.365e1), 1)"/></out>
    <out><xsl:value-of select="round(xs:float(-1.365e1), 1)"/></out>
    <out><xsl:value-of select="round(xs:float(-2.5e0),0)"/></out>
    <out><xsl:value-of select="round(xs:float(2.5e0),0)"/></out>
    <out><xsl:value-of select="round(xs:float(2.4999e2),0)"/></out>
    <out><xsl:value-of select="round(xs:float(8492.15e1), -1)"/></out>
    <out><xsl:value-of select="round(xs:float(8495.551e2),-1)"/></out>
    <out><xsl:value-of select="round(xs:float(8496.787e2),-2)"/></out>
    <out><xsl:value-of select="round(xs:float(45e100),-101)"></xsl:value-of></out>
    <out><xsl:value-of select="round(xs:float(9.9e-99),99)"></xsl:value-of></out>
    <out><xsl:value-of select="round(xs:float('NaN'),0)"/></out>
    <out><xsl:value-of select="round(xs:float(1e0 div 0e0), 0)"/></out>
   <out>
    <xsl:value-of select="round(xs:float(sum(for $e in a return xs:float($e))), 2)"/>   
   </out>
  </x>
 </xsl:template>
</xsl:stylesheet>
