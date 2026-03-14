<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <xsl:key name="mykey" match="div" use="title"/>

  <xsl:template match="doc">
  <out>
      <!--Should say "Intro Section.":--> 
      <P><xsl:value-of select="key('mykey', 'Introduction')/p"/></P>
      <!--Should say "SS Section.":--> 
      <P><xsl:value-of select="key('mykey', 'Stylesheet Structure')/p"/></P>
      <!--Should say "Exp Section.":--> 
      <P><xsl:value-of select="key('mykey', 'Expressions')/p"/></P>
  </out>    
  </xsl:template>
    
</xsl:stylesheet>