<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec fo#func-number?>
<!-- BUG (reported but not reproduced) that number('123abc') returns 123
     rather than NaN. Appears to depend on the Java VM in use. -->

<xsl:template match="/">
  <testing>
    <xsl:value-of select="number('123abc')"/>
  </testing>
</xsl:template>

</xsl:stylesheet>
