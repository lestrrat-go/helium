<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#constructing-sequences?>
<?error XTSE0010?>
<!-- PURPOSE:   Test sequence construction using xsl:sequence with content -->
<!-- ERROR: xsl:sequence cannot have content -->

<xsl:output indent="yes"/>

<xsl:template match="doc">
  <xsl:variable name="q" as="xs:integer *" xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xsl:for-each select="1 to 3">
       <xsl:sequence>
          <e>5</e>
          <xsl:sequence select=".*10"/>
          <f>6</f>
          <xsl:sequence select=".*10 + 1"/>
          <g>7</g>
       </xsl:sequence>
    </xsl:for-each>
  </xsl:variable>
  <zzz><xsl:value-of select="$q" separator=" "/></zzz>
</xsl:template>
 
</xsl:stylesheet>
