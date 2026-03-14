<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:my="http://www.example.com/my-functions" exclude-result-prefixes="my">

  <!-- PURPOSE test xsl:evaluate cannot call fn:document() -->


  <xsl:param name="p">document('http://www.w3.org')</xsl:param>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:evaluate xpath="$p"/>
    </out>
  </xsl:template>

 
</xsl:stylesheet>
