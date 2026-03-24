<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test xsl:number level="any" where the a node matches both the from and count patterns -->

<!-- The result of this test is surprising! See bug 5849 -->

<xsl:template match="*">
  <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:attribute name="nr">
      <xsl:number from="a[@mark='true']" count="a" level="any"/>
    </xsl:attribute>
    <xsl:apply-templates/>
  </xsl:copy>
</xsl:template>
  
<xsl:template match="text()"/>  
 
</xsl:stylesheet>
