<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 3.0 with XPath 3.1: test format-time: basic numeric formats -->

  <xsl:param name="t" as="xs:time" select="xs:time('09:15:06.456')"/>

  <xsl:template name="main">
    <out>
      <tr a="09:15" x='{format-time($t,"[H01]:[m01]")}'/>
      <tr a="9:15" x='{format-time($t,"[H]:[m]")}'/>
      <tr a="09:15:06" x='{format-time($t,"[H01]:[m01]:[s01]")}'/>
      <tr a="9:15:06" x='{format-time($t,"[H]:[m]:[s]")}'/>
      <tr a="9:15:6" x='{format-time($t,"[H]:[m]:[s1]")}'/>
      <tr a="9:15:06.4" x='{format-time($t,"[H]:[m]:[s].[f,1-1]")}'/>
      <tr a="9:15:06.4" x='{format-time($t,"[H]:[m]:[s].[f1,1-1]")}'/>
      <tr a="9:15:06.45" x='{format-time($t,"[H]:[m]:[s].[f01]")}'/>
      <tr a="9:15:06.456" x='{format-time($t,"[H]:[m]:[s].[f001]")}'/>
    </out>
  </xsl:template>

</xsl:stylesheet>
