<?xml version="1.0" encoding="UTF-8" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
version="2.0">


    <xsl:template match="/">
    <out>
      <xsl:variable name="x">
        <a>
          <x/>
        </a>
        <b>
          <x/>
        </b>
        <c>
          <x/>
        </c>
      </xsl:variable>
      <xsl:for-each select="$x//node()">
        <v><xsl:value-of select="concat('element: ', name(), '; preceding: ', count(preceding::*))"/></v>
      </xsl:for-each>
    </out>  
    </xsl:template>

</xsl:stylesheet>
