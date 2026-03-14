<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#initiating?>
<!-- PURPOSE:   Test use of an initial mode (see config file)  -->


<xsl:template match="doc" mode="inimode">
  <xsl:variable name="q" as="xs:integer *" xmlns:xs="http://www.w3.org/2001/XMLSchema">
    <xsl:for-each select="1 to 10">
       <e><xsl:value-of select="."/></e>
    </xsl:for-each>
  </xsl:variable>
  <zzz><xsl:value-of select="$q" separator=","/></zzz>
</xsl:template>
 
</xsl:stylesheet>
