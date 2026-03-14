<?xml version="1.0" encoding="iso-8859-1"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
  <!-- this shows a bug in Saxon 6.5: duplicate entries in namespace axis
     when the source document explicitly declares the "xml" namespace  -->
  <!-- (but note, some XML parsers don't allow the XML namespace to be declared anyway) -->
  
  <xsl:template match="doc">
    <d>
      <xsl:for-each select="namespace::*">
        <namespace>
          <prefix>
            <xsl:copy-of select="local-name()"/>
          </prefix>
          <uri>
            <xsl:copy-of select="string()"/>
          </uri>
          <name>
            <xsl:copy-of select="name()"/>
          </name>
        </namespace>
      </xsl:for-each>
    </d>
  </xsl:template>
</xsl:stylesheet>
