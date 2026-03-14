<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:apply-import inside an xsl:document while the 
  				imported stylesheet itself is creating a new document using the xsl:document. 
  				The test case is basically testing nested xsl:documents using the xsl:apply-imports. -->

   <t:import href="import1.xsl"/>

   <t:template match="doc">
		    <t:document>
			      <t:apply-imports/>
		    </t:document>
	  </t:template>
</t:transform>
