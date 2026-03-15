<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the value of
                     the use-attribute-sets attribute of an xsl:copy,
                        xsl:element, or xsl:attribute-set element,
                     or the xsl:use-attribute-sets attribute of a literal result element, is not a
                     whitespace-separated sequence of EQNames, or
                     if it contains a QName that does not match the name attribute of
                     any xsl:attribute-set declaration in the stylesheet.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0710?>



  <xsl:template match="/">
      <out>
        <xsl:apply-templates/>
        <xsl:element name="e" use-attribute-sets="#z"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
