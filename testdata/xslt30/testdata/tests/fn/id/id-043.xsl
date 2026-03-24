<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- basic test of idref() with more-than-one as argument -->


  <xsl:template match="/">
    <out>
      <xsl:for-each select="'a', 'b', 'c', 'd'">
          <idref for="{.}">
            <xsl:attribute name="from">
              <xsl:value-of select="for $n in idref(('a', .), doc('id302.xml')) return $n/../@id" separator=","/>
            </xsl:attribute>
          </idref>
      </xsl:for-each>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
