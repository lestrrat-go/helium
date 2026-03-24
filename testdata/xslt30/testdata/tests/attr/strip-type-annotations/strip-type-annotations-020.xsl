<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                input-type-annotations="preserve"
                version="2.0">
<!-- Purpose: Show that when  @input-type-annotations="preserve", the typed value of the 
  				integer 003  is different that its string value.-->

   <xslt:template match="/doc-striptype">     
      <out>
         <xslt:value-of select="
            data(elem-integer),
            string(elem-integer),
            data(elem-integer) instance of xs:integer,
            data(elem-integer) instance of xs:untypedAtomic"
         separator="|"/>
      </out>
   </xslt:template>
</xslt:transform>
