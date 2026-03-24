<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:f="f.uri"
xmlns:fn="http://www.w3.org/2005/xpath-functions"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
exclude-result-prefixes="f fn xs">

<!--  test calling current() within xsl:evaluate -->

<xsl:param name="e1">$p1/*[@n = current()]</xsl:param>

<xsl:variable name="doc">
  <a n="1"/><a n="2"/><a n="3"/><a n="4"/>
</xsl:variable>
    
  <xsl:template match="/" name="main">
    <out>
      <xsl:for-each select="1 to 4">
        <z><xsl:evaluate xpath="$e1" context-item=".">
             <xsl:with-param name="p1" select="$doc"/>
           </xsl:evaluate>
        </z>
      </xsl:for-each>  
    </out>
      
  </xsl:template>
  
 
</xsl:stylesheet>
