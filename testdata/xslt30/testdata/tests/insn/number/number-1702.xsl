<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of current() in the from pattern of xsl:number -->

<xsl:template match="*">
  <xsl:copy>
    <xsl:attribute name="nr">
      <!-- count distance from the last node having the same name as its parent -->
      <xsl:number from="*[name()=name(current())]/*" level="any"/>
    </xsl:attribute>
    <xsl:apply-templates/>
  </xsl:copy>
</xsl:template>
 
</xsl:stylesheet>
