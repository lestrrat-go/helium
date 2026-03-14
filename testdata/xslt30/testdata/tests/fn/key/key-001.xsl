<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test generate-id() as used in grouping. -->

  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <xsl:key name="places" match="town" use="@state"/>

  <xsl:template match="doc">
    <out>
      <xsl:for-each select="town[generate-id() = generate-id(key('places',@state)[1])]">
        <xsl:text>&#10;</xsl:text>
        <xsl:element name="towns">
          <xsl:copy-of select="@state"/>
          <xsl:for-each select="key('places',current()/@state)">
            <xsl:value-of select="@name"/>
            <xsl:if test="position() != last()">
              <xsl:text>, </xsl:text>
            </xsl:if>
          </xsl:for-each>
        </xsl:element>
      </xsl:for-each>
      <xsl:text>&#10;</xsl:text>
    </out>
  </xsl:template>

</xsl:stylesheet>
