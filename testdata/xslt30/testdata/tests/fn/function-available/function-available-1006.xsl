<?xml version="1.0"?> 

<?spec xslt#stylesheet-functions?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:fn = "http://www.w3.org/2003/05/xpath-functions">

<!-- test function-available() using the fn:namespace -->
<?category function-available?>

<xsl:param name="p" select="'fn:implicit-timezone'"/>
<xsl:param name="q" select="'fn:explicit-timezone'"/>

  <xsl:template match="/">
      <out>;
        <xsl:value-of select="function-available('fn:implicit-timezone')"/>;
        <xsl:value-of select="function-available('fn:explicit-timezone')"/>;
        <xsl:value-of select="function-available($p)"/>; 
        <xsl:value-of select="function-available($q)"/>;                        
      </out>
  </xsl:template>


</xsl:stylesheet>
