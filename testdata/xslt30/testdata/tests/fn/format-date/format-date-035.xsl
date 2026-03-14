<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0" exclude-result-prefixes="xs">
  <?spec xslt#format-dateTime?>
  <!-- PURPOSE: XSLT 2.0: test format-dateTime: Error XTDE1340 incorrect picture string-->
  <xsl:param name="d" as="xs:dateTime" select="xs:dateTime('2003-12-07T09:15:06.456')"/>
  <xsl:template name="main">
    <out>;
      <z><xsl:value-of select="format-dateTime($d, '[bla]', 'en', (), ())"/></z>;
    </out>
  </xsl:template>
</xsl:stylesheet>