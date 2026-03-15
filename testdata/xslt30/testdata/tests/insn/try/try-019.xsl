<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" exclude-result-prefixes="err">

  <!-- xsl:catch with variables to receive error information in select attribute -->

  <xsl:template name="main">
    <out>
      <xsl:try rollback-output="yes">
        <xsl:sequence select="doc('rubbish.xml')"/>
        <xsl:catch errors="*"
          select="'Error:', $err:code, $err:description, $err:value, tokenize($err:module, '/')[last()], ('#'||$err:line-number||'#'), $err:column-number"
        />
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>
