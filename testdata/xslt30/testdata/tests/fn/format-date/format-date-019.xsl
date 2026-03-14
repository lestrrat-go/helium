<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0" exclude-result-prefixes="xs">
  <?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-date: Error XTDE1340 incorrect picture string-->
  <xsl:param name="d" as="xs:date" select="xs:date('2003-12-07')"/>
  <xsl:template name="main">
    <out>;
      <z><xsl:value-of select="format-date($d, '[bla]', 'en', (), ())"/></z>;
    </out>
  </xsl:template>
</xsl:stylesheet>