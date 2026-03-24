<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: COPY41 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 -->
    <!-- Purpose: Try to copy a comment before there is any element produced. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <xsl:copy-of select="doc/comment()"/>
  <out/>
</xsl:template>

</xsl:stylesheet>