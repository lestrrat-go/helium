<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test xsl:key with union in "use" attribute. -->

  <xsl:key name="key1" match="div" use="x | y | z"/>

  <xsl:template match="/">
    <out>
      <group k="25">
        <xsl:for-each select="key('key1', '25')">
          <title>
            <xsl:value-of select="title"/>
          </title>
        </xsl:for-each>
      </group>
      <group k="39">
        <xsl:for-each select="key('key1', '39')">
          <title>
            <xsl:value-of select="title"/>
          </title>
        </xsl:for-each>
      </group>
      <group k="44">
        <xsl:for-each select="key('key1', '44')">
          <title>
            <xsl:value-of select="title"/>
          </title>
        </xsl:for-each>
      </group>
      <group k="75">
        <xsl:for-each select="key('key1', '75')">
          <title>
            <xsl:value-of select="title"/>
          </title>
        </xsl:for-each>
      </group>
    </out>
  </xsl:template>

</xsl:stylesheet>
