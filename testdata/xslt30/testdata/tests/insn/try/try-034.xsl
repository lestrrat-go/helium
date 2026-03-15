<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes="err">

  <!-- xsl:try with rollback=no: a case where catching the error probably fails because lots of output has been written -->

  <xsl:template name="main">
    <out>
      <xsl:try rollback-output="no">
        <xsl:for-each select="1 to 1000">
          <in/>
        </xsl:for-each>
        <xsl:source-document href="rubbish.xml">
          <xsl:value-of select="count(//*)"/>
        </xsl:source-document>
        <xsl:catch errors="*">
          <xsl:text>Error {$err:code} was caught!</xsl:text>
        </xsl:catch>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>
