<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
        xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
        xmlns:d="http://www.kitsite.com/v1/Container/DefaultAttribute"
        xmlns:c="http://www.kitsite.com/v1/Container/Attribute">

<?spec xpath#node-tests?>
<!-- TEST namespace test on attribute axis (bug Saxon 6.2.2/002) -->

  <xsl:template match="XXX">
    <foo>
      <xsl:copy-of select="@c:*"/>
    </foo>
  </xsl:template>


</xsl:stylesheet>
