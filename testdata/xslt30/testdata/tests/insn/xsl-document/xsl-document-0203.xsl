<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests an LRE which contains a document node and an xsl:element instruction 
  				whose sequence constructor contains a document node. -->

   <t:template match="/">
		    <out>
         <t:document>
				        <t:element name="Top">
					          <t:document>
						            <a/>
						            <b/>
					          </t:document>
				        </t:element>
			      </t:document>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
