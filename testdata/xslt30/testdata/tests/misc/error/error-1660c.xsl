<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet 
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!--A non-schema-aware processor must signal a static
       error if the stylesheet includes an [xsl:]type attribute, or an
       [xsl:]validation or default-validation attribute
       with a value other than strip, preserve, or lax.-->
   <xsl:template name="main" >
      <xsl:element validation="strict">
         <x/>
      </xsl:element>
   </xsl:template>
</xsl:stylesheet>
