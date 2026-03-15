<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs err">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>
  
  <xsl:variable name="extra" as="element(ITEM)">
    <ITEM CAT="H">
      <TITLE>Ulysses</TITLE>
      <AUTHOR>James Joyce</AUTHOR>
      <PUBLISHER>HarperCollins</PUBLISHER>
      <PUB-DATE>1935-06-02</PUB-DATE>
      <LANGUAGE>English</LANGUAGE>
      <PRICE>18.90</PRICE>
      <QUANTITY>235</QUANTITY>		
      <ISBN>0186701805</ISBN>
      <PAGES>830</PAGES>
      <DIMENSIONS UNIT="in">7.2 5.6 1.2</DIMENSIONS>
      <WEIGHT UNIT="oz">11.0</WEIGHT>
    </ITEM>
  </xsl:variable>


  <!-- within xsl:source-document, use xsl:fork/xsl:for-each-group -->

  <xsl:template name="g-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@CAT">
            <CAT ID="{current-grouping-key()}">
              <xsl:copy-of select="current-group()"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:fork/xsl:for-each-group with a grounded population -->

  <xsl:template name="g-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM/copy-of()" group-by="@CAT">
            <CAT ID="{current-grouping-key()}">
              <xsl:copy-of select="current-group()"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:fork/xsl:for-each-group/xsl:result-document -->
  
  <xsl:template name="g-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@CAT">
            <xsl:result-document href="{current-grouping-key()}.xml">
              <CAT ID="{current-grouping-key()}">
                <xsl:copy-of select="current-group()"/>
              </CAT>
            </xsl:result-document>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- Aggregation over current-group -->
  
  <xsl:template name="g-004" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@CAT">
            <CAT ID="{current-grouping-key()}" SIZE="{count(current-group())}"/>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- Non-consuming action -->
  
  <xsl:template name="g-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@CAT">
            <CAT ID="{current-grouping-key()}"/>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:for-each-group with both striding and grounded items -->
  
  <xsl:template name="g-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="(/BOOKLIST/BOOKS/ITEM, $extra)" group-by="@CAT">
            <CAT ID="{current-grouping-key()}">
              <xsl:copy-of select="current-group()"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:for-each-group with both striding and grounded items -->
  
  <xsl:template name="g-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="($extra, /BOOKLIST/BOOKS/ITEM)" group-by="@CAT">
            <CAT ID="{current-grouping-key()}">
              <xsl:copy-of select="current-group()"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:for-each-group downward selection from current-group(), mixed posture -->
  
  <xsl:template name="g-008" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="($extra, /BOOKLIST/BOOKS/ITEM)" group-by="@CAT">
            <CAT ID="{current-grouping-key()}">
              <xsl:copy-of select="current-group()!PRICE"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:for-each-group downward selection from current-group(), mixed posture -->
  <!-- Note: see bug 29507 -->
  
  <xsl:template name="g-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="($extra, /BOOKLIST/BOOKS/ITEM)" group-by="@CAT">
            <CAT ID="{current-grouping-key()}">
              <xsl:copy-of select="current-group() / PRICE"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- streamed xsl:for-each-group with composite key -->
  
  <xsl:template name="g-010" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@PUB-DATE, @LANGUAGE" composite="yes">
            <CAT DATE="{current-grouping-key()[1]}" LANG="{current-grouping-key()[2]}"
              AVG-PRICE="{round(avg(current-group() / @PRICE), 3)}">
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- streamed xsl:for-each-group with composite key (context item is first in group) -->
  
  <xsl:template name="g-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@AUTHOR, @PRICE" composite="true">
            <xsl:variable name="group" select="copy-of(current-group())"/>
            <CAT ID="{distinct-values($group/@CAT)}" AUTHOR="{@AUTHOR}" PRICE="{@PRICE}"
              COUNT="{count($group)}"
              AVG-QTY="{round(avg($group / @QUANTITY), 3)}">
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- streamed xsl:for-each-group: position of groups  -->
  
  <xsl:template name="g-012" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@AUTHOR">
            <g author="{current-grouping-key()}" position="{position()}"/>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- streamed xsl:for-each-group: current-grouping-key not available in called template -->
  
  <xsl:template name="g-013" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@AUTHOR">
            <g author="{current-grouping-key()}">
              <xsl:call-template name="cgk"/>
            </g>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template name="cgk">
    <xsl:context-item use="absent"/>
    <xsl:try>
      <h key="{current-grouping-key()}"/>
      <xsl:catch errors="*:XTDE1071">
        <h key="#absent#"/>
      </xsl:catch>
    </xsl:try>  
  </xsl:template>
  
  <!-- streamed xsl:for-each-group: current-group not available in called template -->
  
  <xsl:template name="g-014" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="@AUTHOR">
            <g author="{current-grouping-key()}">
              <xsl:call-template name="cg"/>
            </g>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template name="cg">
    <xsl:context-item use="absent"/>
    <xsl:try>
      <h size="{count(current-group())}"/>
      <xsl:catch errors="*:XTDE1061">
        <h size="#absent#"/>
      </xsl:catch>
    </xsl:try>  
  </xsl:template>
  
  
  <!-- streamed xsl:for-each-group: items can be in mode than one group -->
  
  <xsl:template name="g-017" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books-atts.xml">
        <xsl:fork>
          <xsl:for-each-group select="/BOOKLIST/BOOKS/ITEM" group-by="string-to-codepoints(@AUTHOR) ! codepoints-to-string(.)" composite="no">
            <g author-contains="{current-grouping-key()}">
              <books>
                <xsl:value-of select="current-group()/@TITLE" separator="|"/>
              </books>
            </g>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- streamed xsl:for-each-group: items can be in mode than one group -->
  
  <xsl:template name="g-018" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:fork>
          <xsl:for-each-group select="($extra, /BOOKLIST/BOOKS/ITEM)" group-by="string-to-codepoints(@CAT) ! codepoints-to-string(.)" composite="no">
            <CAT ID="{current-grouping-key()}">
              <xsl:value-of select="current-group() / TITLE" separator="|"/>
            </CAT>
          </xsl:for-each-group>
        </xsl:fork>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  



</xsl:transform>
