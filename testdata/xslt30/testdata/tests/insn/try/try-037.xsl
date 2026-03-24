<?xml version="1.0"?> 

<!-- try007 -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:err="http://www.w3.org/2005/xqt-errors" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs err">

  <!-- Catch no-namespace error code -->


  <xsl:template name="main" xpath-default-namespace="http://www.w3.org/2005/xqt-errors">
    <out>
      <xsl:try select="if (current-date() gt xs:date('1900-01-01')) then error(QName('', 'too-late')) else 'too-early'">
        <xsl:catch errors="too-late" select="'Caught'"/>
        <xsl:catch errors="*" select="'Fail'"/>
      </xsl:try>
    </out>
  </xsl:template>

</xsl:stylesheet>
