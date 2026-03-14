<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0" exclude-result-prefixes="xs">
  <?spec xslt#format-time?>
  <!-- PURPOSE: XSLT 2.0: test format-time: Error XTDE1350 component in picture string not available in value-->
  <xsl:param name="d" as="xs:time" select="xs:time('09:15:06.456')"/>
  <xsl:template name="main">
    <out>;
      <z><xsl:value-of select="format-time($d, '[M]', 'en', (), ())"/></z>;
    </out>
  </xsl:template>
</xsl:stylesheet>