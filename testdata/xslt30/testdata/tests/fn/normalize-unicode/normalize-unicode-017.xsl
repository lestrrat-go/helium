<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <xsl:output indent="no" encoding="iso-8859-1"/>
  <xsl:strip-space elements="doc"/>
  <!-- trivial test of normalization-form="none" -->
  <?spec ser#normalization-form?>

  <xsl:template match="/">
    <xsl:result-document method="xml" normalization-form="none">
      <html>
        <xsl:copy-of select="."/>
      </html>
    </xsl:result-document>
  </xsl:template>


</xsl:stylesheet>
