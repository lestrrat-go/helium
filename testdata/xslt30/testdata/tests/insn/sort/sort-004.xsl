<?xml version="1.0"?> 

<!-- A test for xsl:sort, calling a function within a sequence constructor -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:f="http://s.com/" exclude-result-prefixes="f">

  <xsl:template match="doc">
    <out>
      <asc>
        <xsl:for-each select="item">
          <xsl:sort>
            <xsl:sequence select="f:reverse(.)"/>
          </xsl:sort>
          <xsl:copy-of select="."/>
          <xsl:text> </xsl:text>
        </xsl:for-each>
      </asc>
      <desc>
        <xsl:for-each select="item">
          <xsl:sort order="descending">
            <xsl:sequence select="f:reverse(.)"/>
          </xsl:sort>
          <xsl:copy-of select="."/>
          <xsl:text> </xsl:text>
        </xsl:for-each>
      </desc>
    </out>
  </xsl:template>

  <xsl:function name="f:reverse">
    <xsl:param name="s"/>
    <xsl:sequence select="codepoints-to-string(reverse(string-to-codepoints($s)))"/>
  </xsl:function>

</xsl:stylesheet>
