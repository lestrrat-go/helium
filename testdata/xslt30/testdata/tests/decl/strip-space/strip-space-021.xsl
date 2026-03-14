<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- no confict reported in 2.0 if there are two 
       identical strip-space or preserve-space declarations -->
        <?spec xslt#strip?>

  <xsl:output method="xml" indent="no" />

  <xsl:strip-space elements="document"/>
  <xsl:strip-space elements="document"/>

  <xsl:template match="@*|node()">
    <xsl:copy>
      <xsl:apply-templates select="@*|node()"/>
    </xsl:copy>
  </xsl:template>
</xsl:stylesheet>
