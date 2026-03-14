<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                        xsl:sort element other than the first in a sequence of
                     sibling xsl:sort elements has a stable
                     attribute.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1017?>


  <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:sort select="."/>
            <xsl:sort select="number(.)" stable="no"/>
            <xsl:value-of select="."/>
         </xsl:for-each>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
