<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:my="http://www.example.com/my-functions" exclude-result-prefixes="my">

  <!-- PURPOSE A dynamic call to fn:document() may or may not succeed -->


  <xsl:param name="p">function-lookup(QName('http://www.w3.org/2005/xpath-functions', 'document'), 1)
    ('https://www.saxonica.com/welcome/welcome.xml')</xsl:param>
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:evaluate xpath="$p"/>
    </out>
  </xsl:template>

 
</xsl:stylesheet>
