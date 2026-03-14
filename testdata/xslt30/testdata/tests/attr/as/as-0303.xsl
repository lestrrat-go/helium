<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with xsl:template with xsl:value-of on a literal 
  				string inside the sequence constructor and has @as=xs:untypedAtomic.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <out>
         <xslt:apply-templates/>
      </out>
   </xslt:template>

   <xslt:template match="item" as="xs:untypedAtomic">
      <xslt:value-of select="'123'"/>
   </xslt:template>
   
   <xslt:template match="text()"/>
</xslt:transform>
