<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests using forExpr inside an xsl:document.  -->

   <t:variable name="var" select="6"/>

   <t:variable name="v" as="document-node()">
		    <t:document xmlns:xs="http://www.w3.org/2001/XMLSchema"> 
		       <t:value-of select="string-join(for $i in 1 to $var return 'a', 'b')"/>
         <t:text> </t:text>
		       <t:value-of select="avg(for $i in 1 to $var return xs:decimal($i))"/>
         <t:text> </t:text>
		       <t:value-of select="sum(for $i in doc//* return xs:decimal($i))"/>
		    </t:document>
	  </t:variable>

   <t:template match="/">
		    <out>
         <t:copy-of select="$v"/>
      </out>
  	</t:template>

   <t:template match="text()"/>
</t:transform>
