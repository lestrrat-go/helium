<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Sort values that are cast to a specific type dynamically in the select 
  				attribute, but also have data-type to force the real sort to occur on 
  				the string values -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <out>
         <xslt:for-each select="doc/a">
            <xslt:sort select="xs:date(.)" data-type="number"/>
            <a>
               <xslt:value-of select="."/>
            </a>
         </xslt:for-each>
      </out>
   </xslt:template>

   <xslt:template match="text()"/>
</xslt:transform>
