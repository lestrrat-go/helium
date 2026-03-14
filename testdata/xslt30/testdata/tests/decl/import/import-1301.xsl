<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#import?>
  <!-- Purpose: Test of importing a stylesheet involving keys. -->
  <!-- Author: Claudio Sacerdoti Coen, revised by David Marston -->

<!-- Uncomment either line below if keyspace fails when defined only in import. -->
<!-- <xsl:key name="id" use="@id" match="LAMBDA"/> -->
<!-- <xsl:key name="annid" use="@of" match="Annotation"/> -->

<xsl:import href="fragments/imp17all.xsl"/>

</xsl:stylesheet>
