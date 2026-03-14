<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- PURPOSE:   Test sequence construction in a variable -->




  <xsl:template match="doc">
    <out>
      <xsl:variable name="q" as="item() *">
        <xsl:text>a</xsl:text>
        <xsl:attribute name="att">b</xsl:attribute>
        <xsl:comment>c</xsl:comment>
        <xsl:processing-instruction name="pi">data</xsl:processing-instruction>
      </xsl:variable>
      <xsl:for-each select="$q">
        <item>
          <name>
            <xsl:value-of select="name()"/>
          </name>
          <value>
            <xsl:value-of select="."/>
          </value>
          <id>
            <xsl:value-of select="generate-id(.)!=''"/>
          </id>
        </item>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
