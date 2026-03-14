<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: Test overriding of a single attribute of a decimal format,
                 while inheriting other attributes. 
        See Saxon bug http://dev.saxonica.com/community/issues/1571 -->

<xsl:import href="numberformat88a.xsl"/>
<xsl:decimal-format name="q" minus-sign="~"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="format-number(-1931.4857, '0,00', 'q')"/>
  </out>
</xsl:template>

</xsl:stylesheet>
