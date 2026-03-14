<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0" exclude-result-prefixes="xs">
  <?spec xslt#format-dateTime?>
  <!-- PURPOSE: XSLT 2.0: test format-date/Time: trivial cases -->
  <xsl:template name="main">
    <out>
      <xsl:value-of select="format-date(current-date(), '')"/>
      <xsl:value-of select="format-time(current-time(), '  ')"/>
      <xsl:value-of select="format-dateTime(current-dateTime(), '[[QQQ]]')"/>
    </out>
  </xsl:template>
</xsl:stylesheet>