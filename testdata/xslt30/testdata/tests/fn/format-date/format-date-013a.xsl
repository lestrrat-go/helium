<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <?spec xslt#format-date?>
  <!-- PURPOSE: XSLT 2.0: test format-time(): width specifiers -->

  <xsl:param name="t" as="xs:dateTime" select="xs:dateTime('0985-03-01T09:15:06.456')"/>

  <xsl:template name="main">
    <out>

      <year>
        <t a="0985">
          <xsl:value-of select="format-dateTime($t, '[Y,4-4]')"/>
        </t>
        <t a="985">
          <xsl:value-of select="format-dateTime($t, '[Y,3-4]')"/>
        </t>
        <t a="985">
          <xsl:value-of select="format-dateTime($t, '[Y,2-5]')"/>
        </t>
        <t a="85">
          <xsl:value-of select="format-dateTime($t, '[Y,2-2]')"/>
        </t>
        <t a="985">
          <xsl:value-of select="format-dateTime($t, '[Y,2-*]')"/>
        </t>
        <t a="985">
          <xsl:value-of select="format-dateTime($t, '[Y,*-4]')"/>
        </t>
        <t a="985">
          <xsl:value-of select="format-dateTime($t, '[Y,3]')"/>
        </t>
      </year>
      <month>
        <t a="0003">
          <xsl:value-of select="format-dateTime($t, '[M,4-4]')"/>
        </t>
        <t a="3">
          <xsl:value-of select="format-dateTime($t, '[M,1-4]')"/>
        </t>
        <t a="03">
          <xsl:value-of select="format-dateTime($t, '[M,2-5]')"/>
        </t>
        <t a="03">
          <xsl:value-of select="format-dateTime($t, '[M,2-2]')"/>
        </t>
        <t a="3">
          <xsl:value-of select="format-dateTime($t, '[M,1-*]')"/>
        </t>
        <t a="3">
          <xsl:value-of select="format-dateTime($t, '[M,*-2]')"/>
        </t>
        <t a="003">
          <xsl:value-of select="format-dateTime($t, '[M,3]')"/>
        </t>
      </month>
      <micros>
        <t a="4560">
          <xsl:value-of select="format-dateTime($t, '[f,4-4]')"/>
        </t>
        <t a="456">
          <xsl:value-of select="format-dateTime($t, '[f,1-4]')"/>
        </t>
        <t a="456">
          <xsl:value-of select="format-dateTime($t, '[f,2-5]')"/>
        </t>
        <t a="45">
          <xsl:value-of select="format-dateTime($t, '[f,2-2]')"/>
        </t>
        <t a="456">
          <xsl:value-of select="format-dateTime($t, '[f,1-*]')"/>
        </t>
        <t a="45">
          <xsl:value-of select="format-dateTime($t, '[f,*-2]')"/>
        </t>
        <t a="456">
          <xsl:value-of select="format-dateTime($t, '[f,3]')"/>
        </t>
      </micros>

    </out>
  </xsl:template>

</xsl:stylesheet>
