<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes=" err">

  <!-- original purpose: invalid child of xsl:try: xsl:fallback not allowed between xsl:catch children -->
  <!-- but the rules in the spec have changed so this is now allowed. -->

  <xsl:template name="main">
    <out>
      <xsl:try>
        <xsl:sequence select="2+2"/>
        <xsl:catch errors="*"/>
        <xsl:fallback>
          <xsl:sequence select="2+3"/>
        </xsl:fallback>
        <xsl:catch errors="*"/>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>
