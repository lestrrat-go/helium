<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- PURPOSE:   Test sequence construction of parentless elements -->

  <xsl:template match="doc">
    <out>
      <xsl:variable name="q" as="element() *">
        <a>1</a>
        <b>2</b>
        <c>3</c>
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
