<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test namespace attribute of xsl:attribute -->

<xsl:template match="/">
   <out xmlns:anamespace="http://foo.com/">
      <p><xsl:attribute name="test" namespace="http://foo.com/">true</xsl:attribute></p>
      <p><xsl:attribute name="test" namespace="http://baz.com/">true</xsl:attribute></p>
   </out>
</xsl:template>
 
</xsl:stylesheet>
