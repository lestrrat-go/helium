<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#constructing-sequences?>
<?error XTTE0570?>
<!-- PURPOSE:   Test sequence construction using xsl:sequence with a failing type check -->

<xsl:output indent="yes"/>

<xsl:template match="doc">
  <xsl:variable name="q" as="xs:string *" xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xsl:for-each select="1 to 10">
       <xsl:sequence select=".*."/>
    </xsl:for-each>
  </xsl:variable>
  <zzz><xsl:value-of select="$q" separator=","/></zzz>
</xsl:template>
 
</xsl:stylesheet>
