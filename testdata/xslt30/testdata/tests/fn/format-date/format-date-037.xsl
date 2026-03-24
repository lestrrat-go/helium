<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0" exclude-result-prefixes="xs">
  <?spec xslt#format-dateTime?>
  <!-- PURPOSE: Saxon bug 4168: default namespace for calendar attribute -->
  <xsl:template name="main">
    <out xmlns="http://www.idpf.org/2007/opf">
      <xsl:value-of select="format-date(current-date(), '[Y0001]-[M01]-[D01]', 'en', 'AD', ())"/>
    </out>
  </xsl:template>
</xsl:stylesheet>