<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: To output a document with method 'xml' and version '1.1'. 
                When NEL (0x0085) and LSEP (0x2028) appear as Numeric 
                Character Reference (NCR), they must be output as NCR in 
                an attribute node of an xsl:element instruction.  -->

   <t:output method="xml" version="1.1" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:element name="a">
   			      <t:attribute name="name" select="'&#x85;&#x2028;'"/>
    	    </t:element>
      </out>
  </t:template>
</t:transform>
