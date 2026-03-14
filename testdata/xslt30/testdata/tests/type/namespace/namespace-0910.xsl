<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Check the parentage of the namespace nodes for an element -->
  
  <xsl:template match="/">
    <out>
      <xsl:for-each select="//c:chap" xmlns:c="http://chap-default.uri/" exclude-result-prefixes="c">
        <xsl:variable name="p" select="."/>
        <xsl:for-each select="namespace::*">
          <check>
            <xsl:value-of select=".. is $p"/>
          </check>
        </xsl:for-each>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>
