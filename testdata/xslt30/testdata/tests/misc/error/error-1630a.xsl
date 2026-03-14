<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"><!--It is a recoverable dynamic error
                     if an xsl:value-of or xsl:text instruction
                     specifies that output escaping is to be disabled when writing to a final result tree that is not being
                     serialized. The optional recovery action is to ignore the
                           disable-output-escaping attribute.
                  -->
   <xsl:template name="main">
      <out>
         <xsl:value-of select="'&lt;a/&gt;'" disable-output-escaping="yes"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
