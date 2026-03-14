<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Check that exclude-result-prefixes applies to URIs, not prefixes -->
  
  <xsl:template name="main">
    <alpha xmlns:a="a.uri" xmlns:b="b.uri" xmlns:c="c.uri" xsl:exclude-result-prefixes="a c">
      <beta xmlns:a="c.uri" xmlns:b="d.uri" xmlns:c="e.uri" xsl:exclude-result-prefixes="b"/>
    </alpha>
  </xsl:template>

</xsl:stylesheet>
