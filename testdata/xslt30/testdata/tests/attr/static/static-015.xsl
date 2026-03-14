<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <!-- Static param with version attr set, no difference in behavior -->
   
   <xsl:param name="static-param1" static="yes" version="1.0"/>
   <xsl:param name="static-param2" static="yes" version="3.0"/>
   <xsl:param name="static-param3" static="yes" version="345.4"/>

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {$static-param1}{$static-param2}{$static-param3}
   </xsl:template>
   
</xsl:transform>
