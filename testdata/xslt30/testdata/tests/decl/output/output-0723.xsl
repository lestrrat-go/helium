<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
 
   <!-- Serialization of a CR character within a comment node.
        The spec isn't clear what should happen here. The CR can't be escaped.
        If it is output as-is, then it won't round-trip (it will be lost on reparsing).
        But that seems to be the only sensible option -->
 
   <xsl:output method="xml" omit-xml-declaration="yes"/>
   <xsl:template name="xsl:initial-template">
      <a>
         <xsl:comment select="'[' || codepoints-to-string(13) || ']'"/>
      </a>
   </xsl:template>
</xsl:stylesheet>