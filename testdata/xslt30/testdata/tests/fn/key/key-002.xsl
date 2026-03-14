<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <?spec xslt#keys?>
  <!-- test use of third para of key() to select a subtree -->

  <xsl:key name="town" match="town" use="@name"/>

  <xsl:template match="doc">

    <!-- first group the data by state -->

    <xsl:variable name="tree">
      <xsl:for-each-group select="town" group-by="@state">
        <state name="{current-grouping-key()}">
          <xsl:copy-of select="current-group()"/>
        </state>
      </xsl:for-each-group>
    </xsl:variable>

    <out>

      <!-- find Enfield, CT -->

      <ct-only>
        <xsl:copy-of select="key('town', 'Enfield', $tree/state[@name='CT'])"/>
      </ct-only>

      <!-- find all Enfields -->

      <all-states>
        <xsl:copy-of select="$tree/state[@name='CT']/key('town', 'Enfield')"/>
      </all-states>
    </out>
  </xsl:template>

</xsl:stylesheet>
