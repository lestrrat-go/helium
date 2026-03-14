<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests character references with xsl:import and tunnel parameters.
  				Test case uses version="1.0" on the main stylesheet and version="1.1 
  				in the xml declaration of the imported stylesheet.  -->

   <t:import href="IRI_import.xsl"/>

   <t:output method="xml" version="1.1"/>

   <t:template match="/">
  	   <t:apply-imports>
  		     <t:with-param name="param" select="'rosé'" tunnel="yes"/>
  	   </t:apply-imports>
  </t:template>

   <t:template match="text()"/>
</t:transform>
