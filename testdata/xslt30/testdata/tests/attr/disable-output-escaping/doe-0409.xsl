<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns="http://www.w3.org/1999/xhtml" version="3.0">
  <xsl:output method="xml" encoding="UTF-8" indent="no" omit-xml-declaration="yes"/>

  <!-- Purpose: Test for disabling-output-escaping="yes" within xsl:for-each-group -->

<xsl:template name="xsl:initial-template">
  <out>
    <xsl:for-each-group select="1 to 20" group-adjacent=". idiv 5">
      <group>
        <xsl:for-each select="current-group()">
          <xsl:value-of select="'&lt;a&gt;' || . || '&lt;/a&gt;'" disable-output-escaping="yes"/>
        </xsl:for-each>
      </group>
    </xsl:for-each-group>
  </out>
</xsl:template>


</xsl:stylesheet>
