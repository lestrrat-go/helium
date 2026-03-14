<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- test namespace attribute of xsl:element -->

<xsl:template match="/">
   <out xmlns:anamespace="http://foo.com/">
      <p><xsl:element name="test" namespace="http://foo.com/"/></p>
      <p><xsl:element name="test" namespace="http://baz.com/"/></p>
   </out>
</xsl:template>
 
</xsl:stylesheet>
